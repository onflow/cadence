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
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
)

type Value = interpreter.Value

//type MemberAccessibleValue interface {
//	// TODO: See whether `Config` parameter can be removed from the below functions.
//	// Currently it's unknown because `AccountCapabilityControllerValue` members
//	// are not yet implemented.
//
//	GetMember(config *Config, name string) Value
//	SetMember(config *Config, name string, value Value)
//}

//type ResourceKindedValue interface {
//	Value
//	// TODO:
//	//Destroy(interpreter *Interpreter, locationRange LocationRange)
//	//IsDestroyed() bool
//	//isInvalidatedResource(*Interpreter) bool
//	IsResourceKinded() bool
//}
//
//// ReferenceTrackedResourceKindedValue is a resource-kinded value
//// that must be tracked when a reference of it is taken.
//type ReferenceTrackedResourceKindedValue interface {
//	ResourceKindedValue
//	IsReferenceTrackedResourceKindedValue()
//	ValueID() atree.ValueID
//	IsStaleResource() bool
//}

//// IterableValue is a value which can be iterated over, e.g. with a for-loop
//type IterableValue interface {
//	Value
//	Iterator() ValueIterator
//}

//type ValueIterator interface {
//	interpreter.ValueIterator
//	Value
//}

// ConvertAndBox converts a value to a target type, and boxes in optionals and any value, if necessary
// TODO: Remove this and re-use interpreter's method
func ConvertAndBox(
	gauge common.MemoryGauge,
	value Value,
	valueType, targetType bbq.StaticType,
) Value {
	value = convert(gauge, value, valueType, targetType)
	return BoxOptional(gauge, value, targetType)
}

// TODO: Remove this and re-use interpreter's method
func convert(gauge common.MemoryGauge, value Value, valueType, targetType bbq.StaticType) Value {
	if valueType == nil {
		return value
	}

	unwrappedTargetType := UnwrapOptionalType(targetType)

	// if the value is optional, convert the inner value to the unwrapped target type
	if optionalValueType, valueIsOptional := valueType.(*interpreter.OptionalStaticType); valueIsOptional {
		switch value := value.(type) {
		case interpreter.NilValue:
			return value

		case *interpreter.SomeValue:
			if !optionalValueType.Type.Equal(unwrappedTargetType) {
				innerValue := convert(
					gauge,
					value.InnerValue(),
					optionalValueType.Type,
					unwrappedTargetType,
				)
				return interpreter.NewSomeValueNonCopying(gauge, innerValue)
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

// TODO: Remove this and re-use interpreter's method
func Unbox(value Value) Value {
	for {
		some, ok := value.(*interpreter.SomeValue)
		if !ok {
			return value
		}

		value = some.InnerValue()
	}
}

// BoxOptional boxes a value in optionals, if necessary
// TODO: Remove this and re-use interpreter's method
func BoxOptional(
	gauge common.MemoryGauge,
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
		case *interpreter.SomeValue:
			inner = typedInner.InnerValue()

		case interpreter.NilValue:
			// NOTE: nested nil will be unboxed!
			return inner

		default:
			value = interpreter.NewSomeValueNonCopying(gauge, value)
		}

		targetType = optionalType.Type
	}
	return value
}
