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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// NewInclusiveRangeValue constructs an InclusiveRange value with the provided start, end with default value of step.
func NewInclusiveRangeValue(
	interpreter *Interpreter,
	locationRange LocationRange,
	start IntegerValue,
	end IntegerValue,
	rangeType InclusiveRangeStaticType,
) *CompositeValue {
	startComparable, startOk := start.(ComparableValue)
	endInclusiveComparable, endInclusiveOk := end.(ComparableValue)
	if !startOk || !endInclusiveOk {
		panic(errors.NewUnreachableError())
	}

	step := GetValueForIntegerType(1, rangeType.ElementType)
	if startComparable.Greater(interpreter, endInclusiveComparable, locationRange) {
		elemSemaTy := interpreter.MustConvertStaticToSemaType(rangeType.ElementType)
		if _, ok := sema.AllUnsignedIntegerTypesSet[elemSemaTy]; ok {
			panic(InclusiveRangeConstructionError{
				LocationRange: locationRange,
				Message:       fmt.Sprintf("step value cannot be negative for unsigned integer type %s", elemSemaTy),
			})
		}

		negatedStep, ok := step.Negate(interpreter, locationRange).(IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		step = negatedStep
	}

	return NewInclusiveRangeValueWithStep(interpreter, locationRange, start, end, step, rangeType)
}

// NewInclusiveRangeValue constructs an InclusiveRange value with the provided start, end & step.
func NewInclusiveRangeValueWithStep(
	interpreter *Interpreter,
	locationRange LocationRange,
	start IntegerValue,
	end IntegerValue,
	step IntegerValue,
	rangeType InclusiveRangeStaticType,
) *CompositeValue {

	// Validate that the step is non-zero.
	if step.Equal(interpreter, locationRange, GetValueForIntegerType(0, rangeType.ElementType)) {
		panic(InclusiveRangeConstructionError{
			LocationRange: locationRange,
			Message:       "step value cannot be zero",
		})
	}

	// Validate that the sequence is moving towards the end value.
	// If start < end, step must be > 0
	// If start > end, step must be < 0
	// If start == end, step doesn't matter.
	if (start.Less(interpreter, end, locationRange) && step.Less(interpreter, GetValueForIntegerType(0, rangeType.ElementType), locationRange)) ||
		(start.Greater(interpreter, end, locationRange) && step.Greater(interpreter, GetValueForIntegerType(0, rangeType.ElementType), locationRange)) {

		panic(InclusiveRangeConstructionError{
			LocationRange: locationRange,
			Message:       fmt.Sprintf("sequence is moving away from end: %s due to the value of step: %s and start: %s", end, step, start),
		})
	}

	fields := []CompositeField{
		{
			Name:  sema.InclusiveRangeTypeStartFieldName,
			Value: start,
		},
		{
			Name:  sema.InclusiveRangeTypeEndFieldName,
			Value: end,
		},
		{
			Name:  sema.InclusiveRangeTypeStepFieldName,
			Value: step,
		},
	}

	rangeSemaType := getInclusiveRangeSemaType(interpreter, rangeType)

	rangeValue := NewCompositeValueWithStaticType(
		interpreter,
		locationRange,
		nil,
		rangeSemaType.QualifiedString(),
		common.CompositeKindStructure,
		fields,
		common.ZeroAddress,
		rangeType,
	)

	rangeValue.Functions = map[string]FunctionValue{
		sema.InclusiveRangeTypeContainsFunctionName: NewHostFunctionValue(
			interpreter,
			sema.InclusiveRangeContainsFunctionType(
				rangeSemaType.MemberType,
			),
			func(invocation Invocation) Value {
				return rangeContains(
					rangeValue,
					rangeType,
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
			},
		),
	}

	return rangeValue
}

func getInclusiveRangeSemaType(interpreter *Interpreter, rangeType InclusiveRangeStaticType) *sema.InclusiveRangeType {
	return interpreter.MustConvertStaticToSemaType(rangeType).(*sema.InclusiveRangeType)
}

func rangeContains(
	rangeValue *CompositeValue,
	rangeType InclusiveRangeStaticType,
	interpreter *Interpreter,
	locationRange LocationRange,
	needleValue Value,
) BoolValue {
	start := getFieldAsIntegerValue(rangeValue, interpreter, locationRange, sema.InclusiveRangeTypeStartFieldName)
	endInclusive := getFieldAsIntegerValue(rangeValue, interpreter, locationRange, sema.InclusiveRangeTypeEndFieldName)
	step := getFieldAsIntegerValue(rangeValue, interpreter, locationRange, sema.InclusiveRangeTypeStepFieldName)

	needleInteger := convertAndAssertIntegerValue(needleValue)

	var result bool
	result = start.Equal(interpreter, locationRange, needleInteger) ||
		endInclusive.Equal(interpreter, locationRange, needleInteger)

	if !result {
		greaterThanStart := needleInteger.Greater(interpreter, start, locationRange)
		greaterThanEndInclusive := needleInteger.Greater(interpreter, endInclusive, locationRange)

		if greaterThanStart == greaterThanEndInclusive {
			// If needle is greater or smaller than both start & endInclusive, then it is outside the range.
			result = false
		} else {
			// needle is in between start and endInclusive.
			// start + k * step should be equal to needle i.e. (needle - start) mod step == 0.
			diff, ok := needleInteger.Minus(interpreter, start, locationRange).(IntegerValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			result = diff.Mod(interpreter, step, locationRange).Equal(interpreter, locationRange, GetValueForIntegerType(0, rangeType.ElementType))
		}
	}

	return AsBoolValue(result)
}

// Get the provided int64 value in the required staticType.
// Note: Assumes that the provided value fits within the constraints of the staticType.
func GetValueForIntegerType(value int64, staticType StaticType) IntegerValue {
	switch staticType {
	case PrimitiveStaticTypeInt:
		return NewUnmeteredIntValueFromInt64(value)
	case PrimitiveStaticTypeInt8:
		return NewUnmeteredInt8Value(int8(value))
	case PrimitiveStaticTypeInt16:
		return NewUnmeteredInt16Value(int16(value))
	case PrimitiveStaticTypeInt32:
		return NewUnmeteredInt32Value(int32(value))
	case PrimitiveStaticTypeInt64:
		return NewUnmeteredInt64Value(value)
	case PrimitiveStaticTypeInt128:
		return NewUnmeteredInt128ValueFromInt64(value)
	case PrimitiveStaticTypeInt256:
		return NewUnmeteredInt256ValueFromInt64(value)

	case PrimitiveStaticTypeUInt:
		return NewUnmeteredUIntValueFromUint64(uint64(value))
	case PrimitiveStaticTypeUInt8:
		return NewUnmeteredUInt8Value(uint8(value))
	case PrimitiveStaticTypeUInt16:
		return NewUnmeteredUInt16Value(uint16(value))
	case PrimitiveStaticTypeUInt32:
		return NewUnmeteredUInt32Value(uint32(value))
	case PrimitiveStaticTypeUInt64:
		return NewUnmeteredUInt64Value(uint64(value))
	case PrimitiveStaticTypeUInt128:
		return NewUnmeteredUInt128ValueFromUint64(uint64(value))
	case PrimitiveStaticTypeUInt256:
		return NewUnmeteredUInt256ValueFromUint64(uint64(value))

	case PrimitiveStaticTypeWord8:
		return NewUnmeteredWord8Value(uint8(value))
	case PrimitiveStaticTypeWord16:
		return NewUnmeteredWord16Value(uint16(value))
	case PrimitiveStaticTypeWord32:
		return NewUnmeteredWord32Value(uint32(value))
	case PrimitiveStaticTypeWord64:
		return NewUnmeteredWord64Value(uint64(value))
	case PrimitiveStaticTypeWord128:
		return NewUnmeteredWord128ValueFromUint64(uint64(value))
	case PrimitiveStaticTypeWord256:
		return NewUnmeteredWord256ValueFromUint64(uint64(value))

	default:
		panic(errors.NewUnreachableError())
	}
}

func getFieldAsIntegerValue(
	rangeValue *CompositeValue,
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) IntegerValue {
	return convertAndAssertIntegerValue(
		rangeValue.GetField(
			interpreter,
			locationRange,
			name,
		),
	)
}

func convertAndAssertIntegerValue(value Value) IntegerValue {
	integerValue, ok := value.(IntegerValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return integerValue
}
