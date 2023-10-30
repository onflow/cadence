/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"math/big"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
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
	isValue()
	Accept(interpreter *Interpreter, visitor Visitor)
	Walk(interpreter *Interpreter, walkChild func(Value))
	StaticType(interpreter *Interpreter) StaticType
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
	MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string
	IsResourceKinded(interpreter *Interpreter) bool
	NeedsStoreTo(address atree.Address) bool
	Transfer(
		interpreter *Interpreter,
		locationRange LocationRange,
		address atree.Address,
		remove bool,
		storable atree.Storable,
		preventTransfer map[atree.StorageID]struct{},
	) Value
	DeepRemove(interpreter *Interpreter)
	// Clone returns a new value that is equal to this value.
	// NOTE: not used by interpreter, but used externally (e.g. state migration)
	// NOTE: memory metering is unnecessary for Clone methods
	Clone(interpreter *Interpreter) Value
	IsImportable(interpreter *Interpreter) bool
}

// ValueIndexableValue

type ValueIndexableValue interface {
	Value
	GetKey(interpreter *Interpreter, locationRange LocationRange, key Value) Value
	SetKey(interpreter *Interpreter, locationRange LocationRange, key Value, value Value)
	RemoveKey(interpreter *Interpreter, locationRange LocationRange, key Value) Value
	InsertKey(interpreter *Interpreter, locationRange LocationRange, key Value, value Value)
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
	GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value
	RemoveMember(interpreter *Interpreter, locationRange LocationRange, name string) Value
	// returns whether a value previously existed with this name
	SetMember(interpreter *Interpreter, locationRange LocationRange, name string, value Value) bool
}

// EquatableValue

type EquatableValue interface {
	Value
	// Equal returns true if the given value is equal to this value.
	// If no location range is available, pass e.g. EmptyLocationRange
	Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool
}

func newValueComparator(interpreter *Interpreter, locationRange LocationRange) atree.ValueComparator {
	return func(storage atree.SlabStorage, atreeValue atree.Value, otherStorable atree.Storable) (bool, error) {
		value := MustConvertStoredValue(interpreter, atreeValue)
		otherValue := StoredValue(interpreter, otherStorable, storage)
		return value.(EquatableValue).Equal(interpreter, locationRange, otherValue), nil
	}
}

// ComparableValue
type ComparableValue interface {
	EquatableValue
	Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue
	LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue
	Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue
	GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue
}

// ResourceKindedValue

type ResourceKindedValue interface {
	Value
	Destroy(interpreter *Interpreter, locationRange LocationRange)
	IsDestroyed() bool
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
	StorageID() atree.StorageID
}

// ContractValue is the value of a contract.
// Under normal circumstances, a contract value is always a CompositeValue.
// However, in the test framework, an imported contract is constructed via a constructor function.
// Hence, during tests, the value is a HostFunctionValue.
type ContractValue interface {
	Value
	SetNestedVariables(variables map[string]*Variable)
}

// CapabilityValue
type CapabilityValue interface {
	atree.Storable
	EquatableValue
	isCapabilityValue()
}

// LinkValue
type LinkValue interface {
	Value
	isLinkValue()
}

// IterableValue is a value which can be iterated over, e.g. with a for-loop
type IterableValue interface {
	Value
	Iterator(interpreter *Interpreter) ValueIterator
}

// ValueIterator is an iterator which returns values.
// When Next returns nil, it signals the end of the iterator.
type ValueIterator interface {
	Next(interpreter *Interpreter) Value
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

// NumberValue
type NumberValue interface {
	ComparableValue
	ToInt(locationRange LocationRange) int
	Negate(*Interpreter, LocationRange) NumberValue
	Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	ToBigEndianBytes() []byte
}

func getNumberValueMember(interpreter *Interpreter, v NumberValue, name string, typ sema.Type, locationRange LocationRange) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.ToStringFunctionType,
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter
				memoryUsage := common.NewStringMemoryUsage(
					OverEstimateNumberStringLength(interpreter, v),
				)
				return NewStringValue(
					interpreter,
					memoryUsage,
					func() string {
						return v.String()
					},
				)
			},
		)

	case sema.ToBigEndianBytesFunctionName:
		return NewHostFunctionValue(
			interpreter,
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					sema.ByteArrayType,
				),
			},
			func(invocation Invocation) Value {
				return ByteSliceToByteArrayValue(
					invocation.Interpreter,
					v.ToBigEndianBytes(),
				)
			},
		)

	case sema.NumericTypeSaturatingAddFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.SaturatingPlus(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)

	case sema.NumericTypeSaturatingSubtractFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.SaturatingMinus(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)

	case sema.NumericTypeSaturatingMultiplyFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.SaturatingMul(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)

	case sema.NumericTypeSaturatingDivideFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.SaturatingDiv(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)
	}

	return nil
}

type IntegerValue interface {
	NumberValue
	BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
}

// BigNumberValue is a number value with an integer value outside the range of int64
type BigNumberValue interface {
	NumberValue
	ByteLength() int
	ToBigInt(memoryGauge common.MemoryGauge) *big.Int
}

type SomeStorable struct {
	gauge    common.MemoryGauge
	Storable atree.Storable
}

var _ atree.Storable = SomeStorable{}

func (s SomeStorable) ByteSize() uint32 {
	return cborTagSize + s.Storable.ByteSize()
}

func (s SomeStorable) StoredValue(storage atree.SlabStorage) (atree.Value, error) {
	value := StoredValue(s.gauge, s.Storable, storage)

	return &SomeValue{
		value:         value,
		valueStorable: s.Storable,
	}, nil
}

func (s SomeStorable) ChildStorables() []atree.Storable {
	return []atree.Storable{
		s.Storable,
	}
}

type ReferenceValue interface {
	Value
	isReference()
	ReferencedValue(interpreter *Interpreter, locationRange LocationRange, errorOnFailedDereference bool) *Value
}
