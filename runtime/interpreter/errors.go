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

var _ errors.InternalError = &unsupportedOperation{}

func (*unsupportedOperation) IsInternalError() {}

func (e *unsupportedOperation) Error() string {
	return fmt.Sprintf(
		"internal error: cannot evaluate unsupported %s operation: %s",
		e.kind.Name(),
		e.operation.Symbol(),
	)
}

// Error is the containing type for all errors produced by the interpreter.
type Error struct {
	Err        error
	Location   common.Location
	StackTrace []Invocation
}

func (e Error) Unwrap() error {
	return e.Err
}

func (e Error) Error() string {
	return e.Err.Error()
}

func (e Error) ChildErrors() []error {
	errs := make([]error, 0, 1+len(e.StackTrace))

	for _, invocation := range e.StackTrace {
		locationRange := invocation.GetLocationRange()
		if locationRange.Location == nil {
			continue
		}

		errs = append(
			errs,
			StackTraceError{
				LocationRange: locationRange,
			},
		)
	}

	return append(errs, e.Err)
}

func (e Error) ImportLocation() common.Location {
	return e.Location
}

type StackTraceError struct {
	LocationRange
}

func (e StackTraceError) Error() string {
	return ""
}

func (e StackTraceError) Prefix() string {
	return ""
}

func (e StackTraceError) ImportLocation() common.Location {
	return e.Location
}

// PositionedError wraps an unpositioned error with position info
//
type PositionedError struct {
	Err error
	ast.Range
}

var _ errors.UserError = PositionedError{}

func (PositionedError) IsUserError() {}

func (e PositionedError) Unwrap() error {
	return e.Err
}

func (e PositionedError) Error() string {
	return e.Err.Error()
}

// NotDeclaredError

type NotDeclaredError struct {
	ExpectedKind common.DeclarationKind
	Name         string
}

var _ errors.UserError = NotDeclaredError{}
var _ errors.SecondaryError = NotDeclaredError{}

func (NotDeclaredError) IsUserError() {}

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

// NotInvokableError

type NotInvokableError struct {
	Value Value
}

var _ errors.UserError = NotInvokableError{}

func (NotInvokableError) IsUserError() {}

func (e NotInvokableError) Error() string {
	return fmt.Sprintf("cannot call value: %#+v", e.Value)
}

// ArgumentCountError

type ArgumentCountError struct {
	ParameterCount int
	ArgumentCount  int
}

var _ errors.UserError = ArgumentCountError{}

func (ArgumentCountError) IsUserError() {}

func (e ArgumentCountError) Error() string {
	return fmt.Sprintf(
		"incorrect number of arguments: expected %d, got %d",
		e.ParameterCount,
		e.ArgumentCount,
	)
}

// TransactionNotDeclaredError

type TransactionNotDeclaredError struct {
	Index int
}

var _ errors.UserError = TransactionNotDeclaredError{}

func (TransactionNotDeclaredError) IsUserError() {}

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

func (ConditionError) IsUserError() {}

func (e ConditionError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("%s failed", e.ConditionKind.Name())
	}
	return fmt.Sprintf("%s failed: %s", e.ConditionKind.Name(), e.Message)
}

// RedeclarationError

type RedeclarationError struct {
	Name string
}

var _ errors.UserError = RedeclarationError{}

func (RedeclarationError) IsUserError() {}

func (e RedeclarationError) Error() string {
	return fmt.Sprintf("cannot redeclare: `%s` is already declared", e.Name)
}

// DereferenceError

type DereferenceError struct {
	LocationRange
}

var _ errors.UserError = DereferenceError{}

func (DereferenceError) IsUserError() {}

func (e DereferenceError) Error() string {
	return "dereference failed"
}

// OverflowError

type OverflowError struct{}

var _ errors.UserError = OverflowError{}

func (OverflowError) IsUserError() {}

func (e OverflowError) Error() string {
	return "overflow"
}

// UnderflowError

type UnderflowError struct{}

var _ errors.UserError = UnderflowError{}

func (UnderflowError) IsUserError() {}

func (e UnderflowError) Error() string {
	return "underflow"
}

// UnderflowError

type DivisionByZeroError struct{}

var _ errors.UserError = DivisionByZeroError{}

func (DivisionByZeroError) IsUserError() {}

func (e DivisionByZeroError) Error() string {
	return "division by zero"
}

// InvalidatedResourceError
//
type InvalidatedResourceError struct {
	LocationRange
}

var _ errors.InternalError = InvalidatedResourceError{}

func (InvalidatedResourceError) IsInternalError() {}

func (e InvalidatedResourceError) Error() string {
	return "internal error: resource is invalidated and cannot be used anymore"
}

// DestroyedResourceError is the error which is reported
// when a user uses a destroyed resource through a reference
//
type DestroyedResourceError struct {
	LocationRange
}

var _ errors.UserError = DestroyedResourceError{}

func (DestroyedResourceError) IsUserError() {}

func (e DestroyedResourceError) Error() string {
	return "resource was destroyed and cannot be used anymore"
}

// ForceAssignmentToNonNilResourceError
//
type ForceAssignmentToNonNilResourceError struct {
	LocationRange
}

var _ errors.UserError = ForceAssignmentToNonNilResourceError{}

func (ForceAssignmentToNonNilResourceError) IsUserError() {}

func (e ForceAssignmentToNonNilResourceError) Error() string {
	return "force assignment to non-nil resource-typed value"
}

// ForceNilError
//
type ForceNilError struct {
	LocationRange
}

var _ errors.UserError = ForceNilError{}

func (ForceNilError) IsUserError() {}

func (e ForceNilError) Error() string {
	return "unexpectedly found nil while forcing an Optional value"
}

// ForceCastTypeMismatchError
//
type ForceCastTypeMismatchError struct {
	ExpectedType sema.Type
	LocationRange
}

var _ errors.UserError = ForceCastTypeMismatchError{}

func (ForceCastTypeMismatchError) IsUserError() {}

func (e ForceCastTypeMismatchError) Error() string {
	return fmt.Sprintf(
		"unexpectedly found non-`%s` while force-casting value",
		e.ExpectedType.QualifiedString(),
	)
}

// TypeMismatchError
//
type TypeMismatchError struct {
	ExpectedType sema.Type
	LocationRange
}

var _ errors.UserError = TypeMismatchError{}

func (TypeMismatchError) IsUserError() {}

func (e TypeMismatchError) Error() string {
	return fmt.Sprintf(
		"type mismatch: expected %s",
		e.ExpectedType.QualifiedString(),
	)
}

// InvalidPathDomainError
//
type InvalidPathDomainError struct {
	ActualDomain    common.PathDomain
	ExpectedDomains []common.PathDomain
	LocationRange
}

var _ errors.UserError = InvalidPathDomainError{}
var _ errors.SecondaryError = InvalidPathDomainError{}

func (InvalidPathDomainError) IsUserError() {}

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

// OverwriteError
//
type OverwriteError struct {
	Address AddressValue
	Path    PathValue
	LocationRange
}

var _ errors.UserError = OverwriteError{}

func (OverwriteError) IsUserError() {}

func (e OverwriteError) Error() string {
	return fmt.Sprintf(
		"failed to save object: path %s in account %s already stores an object",
		e.Path,
		e.Address,
	)
}

// CyclicLinkError
//
type CyclicLinkError struct {
	Address common.Address
	Paths   []PathValue
	LocationRange
}

var _ errors.UserError = CyclicLinkError{}

func (CyclicLinkError) IsUserError() {}

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

// ArrayIndexOutOfBoundsError
//
type ArrayIndexOutOfBoundsError struct {
	Index int
	Size  int
	LocationRange
}

var _ errors.UserError = ArrayIndexOutOfBoundsError{}

func (ArrayIndexOutOfBoundsError) IsUserError() {}

func (e ArrayIndexOutOfBoundsError) Error() string {
	return fmt.Sprintf(
		"array index out of bounds: %d, but size is %d",
		e.Index,
		e.Size,
	)
}

// ArraySliceIndicesError
//
type ArraySliceIndicesError struct {
	FromIndex int
	UpToIndex int
	Size      int
	LocationRange
}

var _ errors.UserError = ArraySliceIndicesError{}

func (ArraySliceIndicesError) IsUserError() {}

func (e ArraySliceIndicesError) Error() string {
	return fmt.Sprintf(
		"slice indices [%d:%d] are out of bounds (size %d)",
		e.FromIndex, e.UpToIndex, e.Size,
	)
}

// InvalidSliceIndexError is returned when a slice index is invalid, such as fromIndex > upToIndex
// This error can be returned even when fromIndex and upToIndex are both within bounds.
type InvalidSliceIndexError struct {
	FromIndex int
	UpToIndex int
	LocationRange
}

var _ errors.UserError = InvalidSliceIndexError{}

func (InvalidSliceIndexError) IsUserError() {}

func (e InvalidSliceIndexError) Error() string {
	return fmt.Sprintf("invalid slice index: %d > %d", e.FromIndex, e.UpToIndex)
}

// StringIndexOutOfBoundsError
//
type StringIndexOutOfBoundsError struct {
	Index  int
	Length int
	LocationRange
}

var _ errors.UserError = StringIndexOutOfBoundsError{}

func (StringIndexOutOfBoundsError) IsUserError() {}

func (e StringIndexOutOfBoundsError) Error() string {
	return fmt.Sprintf(
		"string index out of bounds: %d, but length is %d",
		e.Index,
		e.Length,
	)
}

// StringSliceIndicesError
//
type StringSliceIndicesError struct {
	FromIndex int
	UpToIndex int
	Length    int
	LocationRange
}

var _ errors.UserError = StringSliceIndicesError{}

func (StringSliceIndicesError) IsUserError() {}

func (e StringSliceIndicesError) Error() string {
	return fmt.Sprintf(
		"slice indices [%d:%d] are out of bounds (length %d)",
		e.FromIndex, e.UpToIndex, e.Length,
	)
}

// EventEmissionUnavailableError
//
type EventEmissionUnavailableError struct {
	LocationRange
}

var _ errors.UserError = EventEmissionUnavailableError{}

func (EventEmissionUnavailableError) IsUserError() {}

func (e EventEmissionUnavailableError) Error() string {
	return "cannot emit event: unavailable"
}

// UUIDUnavailableError
//
type UUIDUnavailableError struct {
	LocationRange
}

var _ errors.UserError = UUIDUnavailableError{}

func (UUIDUnavailableError) IsUserError() {}

func (e UUIDUnavailableError) Error() string {
	return "cannot get UUID: unavailable"
}

// TypeLoadingError
//
type TypeLoadingError struct {
	TypeID common.TypeID
}

var _ errors.UserError = TypeLoadingError{}

func (TypeLoadingError) IsUserError() {}

func (e TypeLoadingError) Error() string {
	return fmt.Sprintf("failed to load type: %s", e.TypeID)
}

// MissingMemberValueError

type MissingMemberValueError struct {
	Name string
	LocationRange
}

var _ errors.UserError = MissingMemberValueError{}

func (MissingMemberValueError) IsUserError() {}

func (e MissingMemberValueError) Error() string {
	return fmt.Sprintf("missing value for member `%s`", e.Name)
}

// InvocationArgumentTypeError
//
type InvocationArgumentTypeError struct {
	Index         int
	ParameterType sema.Type
	LocationRange
}

var _ errors.UserError = InvocationArgumentTypeError{}

func (InvocationArgumentTypeError) IsUserError() {}

func (e InvocationArgumentTypeError) Error() string {
	return fmt.Sprintf(
		"invalid invocation with argument at index %d: expected %s",
		e.Index,
		e.ParameterType.QualifiedString(),
	)
}

// MemberAccessTypeError
//
type MemberAccessTypeError struct {
	ExpectedType sema.Type
	LocationRange
}

var _ errors.UserError = MemberAccessTypeError{}

func (MemberAccessTypeError) IsUserError() {}

func (e MemberAccessTypeError) Error() string {
	return fmt.Sprintf(
		"invalid member access: expected %s",
		e.ExpectedType.QualifiedString(),
	)
}

// ValueTransferTypeError
//
type ValueTransferTypeError struct {
	TargetType sema.Type
	LocationRange
}

var _ errors.UserError = ValueTransferTypeError{}

func (ValueTransferTypeError) IsUserError() {}

func (e ValueTransferTypeError) Error() string {
	return fmt.Sprintf(
		"invalid transfer of value: expected %s",
		e.TargetType.QualifiedString(),
	)
}

// ResourceConstructionError
//
type ResourceConstructionError struct {
	CompositeType *sema.CompositeType
	LocationRange
}

var _ errors.UserError = ResourceConstructionError{}

func (ResourceConstructionError) IsUserError() {}

func (e ResourceConstructionError) Error() string {
	return fmt.Sprintf(
		"cannot create resource %s: outside of declaring location %s",
		e.CompositeType.QualifiedString(),
		e.CompositeType.Location.String(),
	)
}

// ContainerMutationError
//
type ContainerMutationError struct {
	ExpectedType sema.Type
	ActualType   sema.Type
	LocationRange
}

var _ errors.UserError = ContainerMutationError{}

func (ContainerMutationError) IsUserError() {}

func (e ContainerMutationError) Error() string {
	return fmt.Sprintf(
		"invalid container update: expected a subtype of '%s', found '%s'",
		e.ExpectedType.QualifiedString(),
		e.ActualType.QualifiedString(),
	)
}

// NonStorableValueError
//
type NonStorableValueError struct {
	Value Value
}

var _ errors.UserError = NonStorableValueError{}

func (NonStorableValueError) IsUserError() {}

func (e NonStorableValueError) Error() string {
	return "cannot store non-storable value"
}

// NonStorableStaticTypeError
//
type NonStorableStaticTypeError struct {
	Type StaticType
}

var _ errors.UserError = NonStorableStaticTypeError{}

func (NonStorableStaticTypeError) IsUserError() {}

func (e NonStorableStaticTypeError) Error() string {
	return fmt.Sprintf(
		"cannot store non-storable static type: %s",
		e.Type,
	)
}

// InterfaceMissingLocation is reported during interface lookup,
// if an interface is looked up without a location
type InterfaceMissingLocationError struct {
	QualifiedIdentifier string
}

var _ errors.UserError = InterfaceMissingLocationError{}

func (InterfaceMissingLocationError) IsUserError() {}

func (e InterfaceMissingLocationError) Error() string {
	return fmt.Sprintf(
		"tried to look up interface %s without a location",
		e.QualifiedIdentifier,
	)
}

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

func (InvalidOperandsError) IsUserError() {}

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

// InvalidPublicKeyError is reported during PublicKey creation, if the PublicKey is invalid.
type InvalidPublicKeyError struct {
	PublicKey *ArrayValue
	Err       error
	LocationRange
}

var _ errors.UserError = InvalidPublicKeyError{}

func (InvalidPublicKeyError) IsUserError() {}

func (e InvalidPublicKeyError) Error() string {
	return fmt.Sprintf("invalid public key: %s, err: %s", e.PublicKey, e.Err)
}

func (e InvalidPublicKeyError) Unwrap() error {
	return e.Err
}

// NonTransferableValueError
//
type NonTransferableValueError struct {
	Value Value
}

var _ errors.UserError = NonTransferableValueError{}

func (NonTransferableValueError) IsUserError() {}

func (e NonTransferableValueError) Error() string {
	return "cannot transfer non-transferable value"
}

// DuplicateKeyInResourceDictionaryError
//
type DuplicateKeyInResourceDictionaryError struct {
	LocationRange
}

var _ errors.UserError = DuplicateKeyInResourceDictionaryError{}

func (DuplicateKeyInResourceDictionaryError) IsUserError() {}

func (e DuplicateKeyInResourceDictionaryError) Error() string {
	return "duplicate key in resource dictionary"
}
