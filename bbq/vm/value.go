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

package vm

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
)

type Value interface {
	isValue()
	StaticType(StaticTypeContext) bbq.StaticType
	Transfer(
		transferContext TransferContext,
		address atree.Address,
		remove bool,
		storable atree.Storable,
	) Value
	String() string
}

type StaticTypeContext interface {
	StorageContext
}

type StorageContext interface {
	interpreter.Storage
	common.MemoryGauge
	TypeConverterContext
}

type TransferContext interface {
	StorageContext
	ReferenceTracker
}

type ReferenceTracker interface {
	TrackReferencedResourceKindedValue(id atree.ValueID, value *EphemeralReferenceValue)
	ReferencedResourceKindedValues(atree.ValueID) map[*EphemeralReferenceValue]struct{}
	ClearReferenceTracking(atree.ValueID)
}

type TypeConverterContext interface {
	Interpreter() *interpreter.Interpreter
}

type MemberAccessibleValue interface {
	// TODO: See whether `Config` parameter can be removed from the below functions.
	// Currently it's unknown because `AccountCapabilityControllerValue` members
	// are not yet implemented.

	GetMember(config *Config, name string) Value
	SetMember(config *Config, name string, value Value)
}

type ResourceKindedValue interface {
	Value
	// TODO:
	//Destroy(interpreter *Interpreter, locationRange LocationRange)
	//IsDestroyed() bool
	//isInvalidatedResource(*Interpreter) bool
	IsResourceKinded() bool
}

// ReferenceTrackedResourceKindedValue is a resource-kinded value
// that must be tracked when a reference of it is taken.
type ReferenceTrackedResourceKindedValue interface {
	ResourceKindedValue
	IsReferenceTrackedResourceKindedValue()
	ValueID() atree.ValueID
	IsStaleResource() bool
}

// IterableValue is a value which can be iterated over, e.g. with a for-loop
type IterableValue interface {
	Value
	Iterator() ValueIterator
}

type ValueIterator interface {
	Value
	HasNext() bool
	Next(config *Config) Value
}

// ConvertAndBox converts a value to a target type, and boxes in optionals and any value, if necessary
func ConvertAndBox(
	value Value,
	valueType, targetType bbq.StaticType,
) Value {
	value = convert(value, valueType, targetType)
	return BoxOptional(value, targetType)
}

func convert(value Value, valueType, targetType bbq.StaticType) Value {
	if valueType == nil {
		return value
	}

	unwrappedTargetType := UnwrapOptionalType(targetType)

	// if the value is optional, convert the inner value to the unwrapped target type
	if optionalValueType, valueIsOptional := valueType.(*interpreter.OptionalStaticType); valueIsOptional {
		switch value := value.(type) {
		case NilValue:
			return value

		case *SomeValue:
			if !optionalValueType.Type.Equal(unwrappedTargetType) {
				innerValue := convert(value.value, optionalValueType.Type, unwrappedTargetType)
				return NewSomeValueNonCopying(innerValue)
			}
			return value
		}
	}

	switch unwrappedTargetType {
	// TODO: add other cases
	default:
		return value
	}
}

func Unbox(value Value) Value {
	for {
		some, ok := value.(*SomeValue)
		if !ok {
			return value
		}

		value = some.value
	}
}

// BoxOptional boxes a value in optionals, if necessary
func BoxOptional(
	value Value,
	targetType bbq.StaticType,
) Value {

	inner := value

	for {
		optionalType, ok := targetType.(*interpreter.OptionalStaticType)
		if !ok {
			break
		}

		switch typedInner := inner.(type) {
		case *SomeValue:
			inner = typedInner.value

		case NilValue:
			// NOTE: nested nil will be unboxed!
			return inner

		default:
			value = NewSomeValueNonCopying(value)
		}

		targetType = optionalType.Type
	}
	return value
}
