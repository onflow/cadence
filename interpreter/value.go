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

package interpreter

import (
	"fmt"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

type Unsigned interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

type TypeConformanceResults map[typeConformanceResultEntry]bool

type typeConformanceResultEntry struct {
	EphemeralReferenceValue *EphemeralReferenceValue
	EphemeralReferenceType  StaticType
}

// SeenReferences is a set of seen references.
//
// NOTE: Do not generalize to map[interpreter.Value],
// as not all values are Go hashable, i.e. this might lead to run-time panics
type SeenReferences map[*EphemeralReferenceValue]struct{}

// NonStorable represents a value that cannot be stored
type NonStorable struct {
	Value Value
}

var _ atree.Storable = NonStorable{}

func (s NonStorable) Encode(_ *atree.Encoder) error {
	//nolint:gosimple
	return NonStorableValueError{
		Value: s.Value,
	}
}

func (s NonStorable) ByteSize() uint32 {
	// Return 1 so that atree split and merge operations don't have to handle special cases.
	// Any value larger than 0 and smaller than half of the max slab size works,
	// but 1 results in fewer number of slabs which is ideal for non-storable values.
	return 1
}

func (s NonStorable) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return s.Value, nil
}

func (NonStorable) ChildStorables() []atree.Storable {
	return nil
}

// Value is the Cadence value hierarchy which is heavily tied to the interpreter and persistent storage,
// and has lots of implementation details.
//
// We do not want to expose those details to users
// (for example, Cadence is used as a library in flow-go (FVM), in the Flow Go SDK, etc.),
// because we want to be able to change the API and implementation details;
// nor do we want to require users to Cadence (the library) to write lots of low-level/boilerplate code (e.g. setting up storage).
//
// To accomplish this, cadence.Value is the "user-facing" hierarchy that is easy to work with:
// simple Go types that can be used without an interpreter or storage.
//
// cadence.Value can be converted to an interpreter.Value by "importing" it with importValue,
// and interpreter.Value can be "exported" to a cadence.Value with ExportValue.
type Value interface {
	atree.Value
	// Stringer provides `func String() string`
	// NOTE: important, error messages rely on values to implement String
	fmt.Stringer
	IsValue()
	Accept(interpreter *Interpreter, visitor Visitor, locationRange LocationRange)
	Walk(interpreter *Interpreter, walkChild func(Value), locationRange LocationRange)
	StaticType(context ValueStaticTypeContext) StaticType
	// ConformsToStaticType returns true if the value (i.e. its dynamic type)
	// conforms to its own static type.
	// Non-container values trivially always conform to their own static type.
	// Container values conform to their own static type,
	// and this function recursively checks conformance for nested values.
	// If the container contains static type information about nested values,
	// e.g. the element type of an array, it also ensures the nested values'
	// static types are subtypes.
	ConformsToStaticType(
		interpreter *Interpreter,
		locationRange LocationRange,
		results TypeConformanceResults,
	) bool
	RecursiveString(seenReferences SeenReferences) string
	MeteredString(context ValueStringContext, seenReferences SeenReferences, locationRange LocationRange) string
	IsResourceKinded(context ValueStaticTypeContext) bool
	NeedsStoreTo(address atree.Address) bool
	Transfer(
		transferContext ValueTransferContext,
		locationRange LocationRange,
		address atree.Address,
		remove bool,
		storable atree.Storable,
		preventTransfer map[atree.ValueID]struct{},
		hasNoParentContainer bool, // hasNoParentContainer is true when transferred value isn't an element of another container.
	) Value
	DeepRemove(
		removeContext ValueRemoveContext,
		hasNoParentContainer bool, // hasNoParentContainer is true when transferred value isn't an element of another container.
	)
	// Clone returns a new value that is equal to this value.
	// NOTE: not used by interpreter, but used externally (e.g. state migration)
	// NOTE: memory metering is unnecessary for Clone methods
	Clone(interpreter *Interpreter) Value
	IsImportable(interpreter *Interpreter, locationRange LocationRange) bool
}

// ValueIndexableValue

type ValueIndexableValue interface {
	Value
	GetKey(context ValueComparisonContext, locationRange LocationRange, key Value) Value
	SetKey(context ContainerMutationContext, locationRange LocationRange, key Value, value Value)
	RemoveKey(context ContainerMutationContext, locationRange LocationRange, key Value) Value
	InsertKey(context ContainerMutationContext, locationRange LocationRange, key Value, value Value)
}

type TypeIndexableValue interface {
	Value
	GetTypeKey(interpreter *Interpreter, locationRange LocationRange, ty sema.Type) Value
	SetTypeKey(interpreter *Interpreter, locationRange LocationRange, ty sema.Type, value Value)
	RemoveTypeKey(interpreter *Interpreter, locationRange LocationRange, ty sema.Type) Value
}

// MemberAccessibleValue

type MemberAccessibleValue interface {
	Value
	GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value
	RemoveMember(interpreter *Interpreter, locationRange LocationRange, name string) Value
	// returns whether a value previously existed with this name
	SetMember(context MemberAccessibleContext, locationRange LocationRange, name string, value Value) bool
}

type ValueComparisonContext interface {
	common.MemoryGauge
	ValueStaticTypeContext
}

var _ ValueComparisonContext = &Interpreter{}

// EquatableValue

type EquatableValue interface {
	Value
	// Equal returns true if the given value is equal to this value.
	// If no location range is available, pass e.g. EmptyLocationRange
	Equal(context ValueComparisonContext, locationRange LocationRange, other Value) bool
}

func newValueComparator(context ValueComparisonContext, locationRange LocationRange) atree.ValueComparator {
	return func(storage atree.SlabStorage, atreeValue atree.Value, otherStorable atree.Storable) (bool, error) {
		value := MustConvertStoredValue(context, atreeValue)
		otherValue := StoredValue(context, otherStorable, storage)
		return value.(EquatableValue).Equal(context, locationRange, otherValue), nil
	}
}

// ComparableValue
type ComparableValue interface {
	EquatableValue
	Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue
	LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue
	Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue
	GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue
}

// ResourceKindedValue

type ResourceKindedValue interface {
	Value
	Destroy(interpreter *Interpreter, locationRange LocationRange)
	IsDestroyed() bool
	isInvalidatedResource(context ValueStaticTypeContext) bool
}

func maybeDestroy(interpreter *Interpreter, locationRange LocationRange, value Value) {
	resourceKindedValue, ok := value.(ResourceKindedValue)
	if !ok {
		return
	}

	resourceKindedValue.Destroy(interpreter, locationRange)
}

// ReferenceTrackedResourceKindedValue is a resource-kinded value
// that must be tracked when a reference of it is taken.
type ReferenceTrackedResourceKindedValue interface {
	ResourceKindedValue
	IsReferenceTrackedResourceKindedValue()
	ValueID() atree.ValueID
	IsStaleResource(*Interpreter) bool
}

// ContractValue is the value of a contract.
// Under normal circumstances, a contract value is always a CompositeValue.
// However, in the test framework, an imported contract is constructed via a constructor function.
// Hence, during tests, the value is a HostFunctionValue.
type ContractValue interface {
	Value
	SetNestedVariables(variables map[string]Variable)
}

// IterableValue is a value which can be iterated over, e.g. with a for-loop
type IterableValue interface {
	Value
	ForEach(
		interpreter *Interpreter,
		elementType sema.Type,
		function func(value Value) (resume bool),
		transferElements bool,
		locationRange LocationRange,
	)
	Iterator(context ValueStaticTypeContext, locationRange LocationRange) ValueIterator
}

// OwnedValue is a value which has an owner
type OwnedValue interface {
	Value
	GetOwner() common.Address
}

type ValueIteratorContext interface {
	common.MemoryGauge
	NumberValueArithmeticContext
}

// ValueIterator is an iterator which returns values.
// When Next returns nil, it signals the end of the iterator.
type ValueIterator interface {
	HasNext() bool
	Next(context ValueIteratorContext, locationRange LocationRange) Value
}

// atreeContainerBackedValue is an interface for values using atree containers
// (atree.Array or atree.OrderedMap) under the hood.
type atreeContainerBackedValue interface {
	Value
	isAtreeContainerBackedValue()
}

func safeAdd(a, b int, locationRange LocationRange) int {
	// INT32-C
	if (b > 0) && (a > (goMaxInt - b)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (b < 0) && (a < (goMinInt - b)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}
	return a + b
}

func safeMul(a, b int, locationRange LocationRange) int {
	// INT32-C
	if a > 0 {
		if b > 0 {
			// positive * positive = positive. overflow?
			if a > (goMaxInt / b) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// positive * negative = negative. underflow?
			if b < (goMinInt / a) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		}
	} else {
		if b > 0 {
			// negative * positive = negative. underflow?
			if a < (goMinInt / b) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// negative * negative = positive. overflow?
			if (a != 0) && (b < (goMaxInt / a)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		}
	}
	return a * b
}

// FixedPointValue is a fixed-point number value
type FixedPointValue interface {
	NumberValue
	IntegerPart() NumberValue
	Scale() int
}

type AuthorizedValue interface {
	GetAuthorization() Authorization
}
